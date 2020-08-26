import { computed, inject, reactive, toRefs, unref, set, watch } from '@vue/composition-api'
import yup from '@/utils/yup'
import i18n from '@/utils/locale'

export const getMetaNamespace = (ns, o) =>
  ns.reduce((xs, x) => (xs && x in xs) ? xs[x] : {}, o)

export const useInputMetaProps = {
  namespace: {
    type: String
  },
  validator: {
    type: Object
  }
}

export const useInputMeta = (props) => {

  const {
    namespace,
    validator
  } = toRefs(props) // toRefs maintains reactivity w/ destructuring

  // defaults (dereferenced)
  let localProps = reactive({ ...props })
  watch(props, (props) => {
    for(let prop in props) {
      set(localProps, prop, props[prop])
    }
  })

  if (unref(namespace)) {
    // use namespace
    const meta = inject('meta', {})
    const namespaceArr = computed(() => unref(namespace).split('.'))
    const namespaceMeta = computed(() => getMetaNamespace(unref(namespaceArr), meta))

    watch(
      namespaceMeta,
      (namespaceMeta) => {
        const {
          min_length: metaMinLength,
          max_length: metaMaxLength,
          min_value: metaMinValue,
          max_value: metaMaxValue,
          pattern: metaPattern,
          placeholder: metaPlaceholder,
          required: metaRequired,
          type: metaType
        } = unref(namespaceMeta)

       // placeholder
        if (metaPlaceholder)
          set(localProps, 'placeholder', metaPlaceholder)

        // validator
        if (!unref(validator)) {
          let schema = yup.string().nullable()

          if (metaRequired)
            schema = schema.required()

          if (metaPattern) {
            const { regex, message } = metaPattern
            const re = new RegExp(`^${regex}$`)
            schema = schema.matches(re, message)
          }

          if (metaMinLength)
            schema = schema.min(metaMinLength)

          if (metaMaxLength)
            schema = schema.max(metaMaxLength)

          if (metaMinValue)
            schema = schema.minAsInt(metaMinValue)

          if (metaMaxValue)
            schema = schema.maxAsInt(metaMaxValue)

          set(localProps, 'validator', schema)
        }

        // type
        switch(metaType) {
          case 'integer':
            set(localProps, 'type', 'number')
            break
          default:
            set(localProps, 'type', 'text')
        }
      },
      { immediate: true }
    )
  }

  return localProps
}
